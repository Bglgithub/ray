# Android 客户端调用示例

这是使用后端代理方案的 Android 客户端示例代码。

## 优势

- ✅ **无需处理签名**：后端自动处理签名
- ✅ **Secret 安全**：Secret 不暴露给客户端
- ✅ **简化开发**：客户端只需发送订单信息
- ✅ **易于维护**：后端更新不影响客户端

## Kotlin 示例代码

### 1. 添加依赖（build.gradle.kts）

```kotlin
dependencies {
    // Retrofit
    implementation("com.squareup.retrofit2:retrofit:2.9.0")
    implementation("com.squareup.retrofit2:converter-gson:2.9.0")
    
    // OkHttp
    implementation("com.squareup.okhttp3:okhttp:4.11.0")
    implementation("com.squareup.okhttp3:logging-interceptor:4.11.0")
}
```

### 2. 数据模型

```kotlin
// CreateInboundRequest.kt（简化版，只需要订单ID和用户ID）
data class CreateInboundRequest(
    val orderId: String,
    val userId: String
)

// CreateInboundResponse.kt
data class CreateInboundResponse(
    val inboundId: Int,
    val port: Int,
    val tag: String
)

// OrderStatusRequest.kt
data class OrderStatusRequest(
    val orderId: String
)

// OrderStatusResponse.kt
data class OrderStatusResponse(
    val orderId: String,
    val status: String,
    val inboundId: Int,
    val paidAt: Long,
    val usedAt: Long
)

// ApiResponse.kt
data class ApiResponse<T>(
    val success: Boolean,
    val msg: String,
    val data: T?
)
```

### 3. API 接口定义

```kotlin
// XUIApiService.kt
import retrofit2.Call
import retrofit2.http.Body
import retrofit2.http.POST

interface XUIApiService {
    
    @POST("/api/v1/inbound/create")
    fun createInbound(
        @Body request: CreateInboundRequest
    ): Call<ApiResponse<CreateInboundResponse>>
    
    @POST("/api/v1/order/status")
    fun getOrderStatus(
        @Body request: OrderStatusRequest
    ): Call<ApiResponse<OrderStatusResponse>>
}
```

### 4. Retrofit 配置

```kotlin
// ApiClient.kt
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import java.util.concurrent.TimeUnit

object ApiClient {
    // 后端代理服务地址（从配置文件或BuildConfig读取）
    private const val BASE_URL = "http://your-backend-proxy.com:8080"
    
    private val loggingInterceptor = HttpLoggingInterceptor().apply {
        level = if (BuildConfig.DEBUG) {
            HttpLoggingInterceptor.Level.BODY
        } else {
            HttpLoggingInterceptor.Level.NONE
        }
    }
    
    private val okHttpClient = OkHttpClient.Builder()
        .addInterceptor(loggingInterceptor)
        .connectTimeout(30, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .writeTimeout(30, TimeUnit.SECONDS)
        .build()
    
    private val retrofit = Retrofit.Builder()
        .baseUrl(BASE_URL)
        .client(okHttpClient)
        .addConverterFactory(GsonConverterFactory.create())
        .build()
    
    val apiService: XUIApiService = retrofit.create(XUIApiService::class.java)
}
```

### 5. 使用示例

```kotlin
// MainActivity.kt 或 ViewModel
import androidx.lifecycle.lifecycleScope
import kotlinx.coroutines.launch
import retrofit2.Call
import retrofit2.Callback
import retrofit2.Response

class MainActivity : AppCompatActivity() {
    
    private lateinit var apiService: XUIApiService
    
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        
        apiService = ApiClient.apiService
        
        // 用户支付成功后调用
        createInboundAfterPayment(
            orderId = "ORDER_123456789",
            userId = "USER_12345",
            protocol = "vmess"
        )
    }
    
    /**
     * 用户支付成功后创建入站配置（简化版）
     */
    private fun createInboundAfterPayment(
        orderId: String,
        userId: String
    ) {
        // 只需要传递订单ID和用户ID，其他参数由后端自动设置
        val request = CreateInboundRequest(
            orderId = orderId,
            userId = userId
        )
        
        apiService.createInbound(request).enqueue(object : Callback<ApiResponse<CreateInboundResponse>> {
            override fun onResponse(
                call: Call<ApiResponse<CreateInboundResponse>>,
                response: Response<ApiResponse<CreateInboundResponse>>
            ) {
                if (response.isSuccessful && response.body()?.success == true) {
                    val inboundData = response.body()?.data
                    if (inboundData != null) {
                        // 创建成功
                        onInboundCreated(inboundData)
                    } else {
                        // 处理错误
                        onError("创建失败: ${response.body()?.msg}")
                    }
                } else {
                    onError("请求失败: ${response.message()}")
                }
            }
            
            override fun onFailure(call: Call<ApiResponse<CreateInboundResponse>>, t: Throwable) {
                onError("网络错误: ${t.message}")
            }
        })
    }
    
    /**
     * 查询订单状态
     */
    private fun checkOrderStatus(orderId: String) {
        val request = OrderStatusRequest(orderId = orderId)
        
        apiService.getOrderStatus(request).enqueue(object : Callback<ApiResponse<OrderStatusResponse>> {
            override fun onResponse(
                call: Call<ApiResponse<OrderStatusResponse>>,
                response: Response<ApiResponse<OrderStatusResponse>>
            ) {
                if (response.isSuccessful && response.body()?.success == true) {
                    val orderData = response.body()?.data
                    if (orderData != null) {
                        when (orderData.status) {
                            "pending" -> showMessage("订单待支付")
                            "paid" -> showMessage("订单已支付，可以创建入站")
                            "used" -> showMessage("订单已使用，入站ID: ${orderData.inboundId}")
                            "expired" -> showMessage("订单已过期")
                        }
                    }
                } else {
                    onError("查询失败: ${response.body()?.msg}")
                }
            }
            
            override fun onFailure(call: Call<ApiResponse<OrderStatusResponse>>, t: Throwable) {
                onError("网络错误: ${t.message}")
            }
        })
    }
    
    private fun onInboundCreated(data: CreateInboundResponse) {
        // 处理创建成功
        Toast.makeText(this, "入站创建成功！端口: ${data.port}", Toast.LENGTH_LONG).show()
        // 保存入站信息，用于后续使用...
    }
    
    private fun onError(message: String) {
        Toast.makeText(this, message, Toast.LENGTH_LONG).show()
    }
    
    private fun showMessage(message: String) {
        Toast.makeText(this, message, Toast.LENGTH_SHORT).show()
    }
}
```

### 6. 使用 Kotlin Coroutines（推荐）

```kotlin
// 使用 suspend 函数和 Coroutines
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import retrofit2.HttpException

suspend fun createInboundSuspend(
    orderId: String,
    userId: String
): Result<CreateInboundResponse> {
    return withContext(Dispatchers.IO) {
        try {
            val request = CreateInboundRequest(
                orderId = orderId,
                userId = userId
            )
            
            val response = apiService.createInbound(request).execute()
            
            if (response.isSuccessful && response.body()?.success == true) {
                val data = response.body()?.data
                if (data != null) {
                    Result.success(data)
                } else {
                    Result.failure(Exception(response.body()?.msg ?: "未知错误"))
                }
            } else {
                Result.failure(Exception(response.body()?.msg ?: "请求失败"))
            }
        } catch (e: HttpException) {
            Result.failure(Exception("HTTP错误: ${e.code()}"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

// 在 ViewModel 中使用
class MainViewModel : ViewModel() {
    
    fun createInbound(orderId: String, userId: String) {
        viewModelScope.launch {
            when (val result = createInboundSuspend(orderId, userId)) {
                is Result.Success -> {
                    // 成功
                    _uiState.value = UiState.Success(result.getOrNull())
                }
                is Result.Failure -> {
                    // 失败
                    _uiState.value = UiState.Error(result.exceptionOrNull()?.message ?: "未知错误")
                }
            }
        }
    }
}
```

## 配置说明

### 开发环境配置

在 `build.gradle.kts` 中配置不同环境的服务器地址：

```kotlin
android {
    buildTypes {
        getByName("debug") {
            buildConfigField("String", "BACKEND_URL", "\"http://192.168.1.100:8080\"")
        }
        getByName("release") {
            buildConfigField("String", "BACKEND_URL", "\"https://api.yourdomain.com\"")
        }
    }
}
```

然后在使用时：

```kotlin
private const val BASE_URL = BuildConfig.BACKEND_URL
```

## 错误处理

```kotlin
sealed class ApiResult<out T> {
    data class Success<out T>(val data: T) : ApiResult<T>()
    data class Error(val message: String, val code: Int? = null) : ApiResult<Nothing>()
    object Loading : ApiResult<Nothing>()
}

suspend fun <T> safeApiCall(
    apiCall: suspend () -> Response<ApiResponse<T>>
): ApiResult<T> {
    return try {
        val response = apiCall()
        if (response.isSuccessful && response.body()?.success == true) {
            val data = response.body()?.data
            if (data != null) {
                ApiResult.Success(data)
            } else {
                ApiResult.Error(response.body()?.msg ?: "数据为空")
            }
        } else {
            ApiResult.Error(
                response.body()?.msg ?: "请求失败",
                response.code()
            )
        }
    } catch (e: Exception) {
        ApiResult.Error("网络错误: ${e.message}")
    }
}

// 使用
viewModelScope.launch {
    _uiState.value = ApiResult.Loading
    _uiState.value = safeApiCall {
        apiService.createInbound(request).execute()
    }
}
```

## 安全建议

1. **使用 HTTPS**：生产环境必须使用 HTTPS
2. **证书锁定**：使用 Certificate Pinning 防止中间人攻击
3. **混淆代码**：启用 ProGuard 或 R8 混淆
4. **不保存敏感信息**：即使使用后端代理，也不要保存不必要的敏感信息

## 完整项目结构

```
app/
├── src/
│   ├── main/
│   │   ├── java/com/yourapp/
│   │   │   ├── api/
│   │   │   │   ├── ApiClient.kt
│   │   │   │   └── XUIApiService.kt
│   │   │   ├── model/
│   │   │   │   ├── CreateInboundRequest.kt
│   │   │   │   ├── CreateInboundResponse.kt
│   │   │   │   └── ...
│   │   │   ├── ui/
│   │   │   │   └── MainActivity.kt
│   │   │   └── viewmodel/
│   │   │       └── MainViewModel.kt
```

## 测试

### 单元测试示例

```kotlin
@Test
fun testCreateInbound() = runTest {
    val mockResponse = ApiResponse(
        success = true,
        msg = "操作成功",
        data = CreateInboundResponse(
            inboundId = 123,
            port = 443,
            tag = "inbound-443"
        )
    )
    
    // 使用 MockWebServer 进行测试
    // ...
}
```

---

**注意**：这是示例代码，实际使用时请根据你的项目结构调整。

